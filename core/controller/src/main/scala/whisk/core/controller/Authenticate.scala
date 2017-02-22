/*
 * Copyright 2015-2016 IBM Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package whisk.core.controller

import scala.concurrent.ExecutionContext
import scala.concurrent.Future
import scala.util.Try

import spray.routing.authentication.UserPass
import whisk.common.Logging
import whisk.common.TransactionId
import whisk.core.database.NoDocumentException
import whisk.core.entity.AuthKey
import whisk.core.entity.Identity
import whisk.core.entity.Secret
import whisk.core.entity.UUID
import whisk.core.entity.WhiskAuthStore
import whisk.core.entity.types.AuthStore

object Authenticate {
    /** Required properties for this component */
    def requiredProperties = WhiskAuthStore.requiredProperties
}

/** A trait to validate basic auth credentials */
trait Authenticate {

    /** An execution context for futures */
    protected implicit val executionContext: ExecutionContext

    /** Database service to lookup credentials */
    protected val authStore: AuthStore

    protected implicit val logging: Logging

    /**
     * Validates credentials against the authentication database; may be used in
     * authentication directive.
     */
    def validateCredentials(userpass: Option[UserPass])(implicit transid: TransactionId): Future[Option[Identity]] = {
        userpass flatMap { pw =>
            Try {
                // authkey deserialization is wrapped in a try to guard against malformed values
                val authkey = AuthKey(UUID(pw.user), Secret(pw.pass))
                logging.info(this, s"authenticate: ${authkey.uuid}")
                val future = Identity.get(authStore, authkey) map { result =>
                    if (authkey == result.authkey) {
                        logging.info(this, s"authentication valid")
                        Some(result)
                    } else {
                        logging.info(this, s"authentication not valid")
                        None
                    }
                } recover {
                    case _: NoDocumentException | _: IllegalArgumentException =>
                        logging.info(this, s"authentication not valid")
                        None
                }
                future onFailure { case t => logging.error(this, s"authentication error: $t") }
                future
            }.toOption
        } getOrElse {
            userpass.foreach(_ => logging.info(this, s"credentials are malformed"))
            Future.successful(None)
        }
    }
}
